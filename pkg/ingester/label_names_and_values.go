// SPDX-License-Identifier: AGPL-3.0-only

package ingester

import (
	"context"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/index"

	"github.com/grafana/mimir/pkg/ingester/client"
)

const checkContextErrorSeriesCount = 1000 // series count interval in which context cancellation must be checked.

// labelNamesAndValues streams the messages with the labels and values of the labels matching the `matchers` param.
// Messages are immediately sent as soon they reach message size threshold defined in `messageSizeThreshold` param.
func labelNamesAndValues(
	index tsdb.IndexReader,
	matchers []*labels.Matcher,
	messageSizeThreshold int,
	server client.Ingester_LabelNamesAndValuesServer,
) error {
	ctx := server.Context()

	labelNames, err := index.LabelNames(matchers...)
	if err != nil {
		return err
	}

	response := client.LabelNamesAndValuesResponse{}
	responseSizeBytes := 0
	for _, labelName := range labelNames {
		if err := ctx.Err(); err != nil {
			return err
		}
		labelItem := &client.LabelValues{LabelName: labelName}
		responseSizeBytes += len(labelName)
		// send message if (response size + size of label name of current label) is greater or equals to threshold
		if responseSizeBytes >= messageSizeThreshold {
			err = client.SendLabelNamesAndValuesResponse(server, &response)
			if err != nil {
				return err
			}
			response.Items = response.Items[:0]
			responseSizeBytes = len(labelName)
		}
		values, err := index.LabelValues(labelName, matchers...)
		if err != nil {
			return err
		}

		lastAddedValueIndex := -1
		for i, val := range values {
			// sum up label values length until response size reached the threshold and after that add all values to the response
			// starting from last sent value or from the first element and up to the current element (including).
			responseSizeBytes += len(val)
			if responseSizeBytes >= messageSizeThreshold {
				labelItem.Values = values[lastAddedValueIndex+1 : i+1]
				lastAddedValueIndex = i
				response.Items = append(response.Items, labelItem)
				err = client.SendLabelNamesAndValuesResponse(server, &response)
				if err != nil {
					return err
				}
				// reset label values to reuse labelItem for the next values of current label.
				labelItem.Values = labelItem.Values[:0]
				response.Items = response.Items[:0]
				if i+1 == len(values) {
					// if it's the last value for this label then response size must be set to `0`
					responseSizeBytes = 0
				} else {
					// if it is not the last value for this label then response size must be set to length of current label name.
					responseSizeBytes = len(labelName)
				}
			} else if i+1 == len(values) {
				// if response size does not reach the threshold, but it's the last label value then it must be added to labelItem
				// and label item must be added to response.
				labelItem.Values = values[lastAddedValueIndex+1 : i+1]
				response.Items = append(response.Items, labelItem)
			}
		}
	}
	// send the last message if there is some data that was not sent.
	if response.Size() > 0 {
		return client.SendLabelNamesAndValuesResponse(server, &response)
	}
	return nil
}

// labelValuesCardinality returns all values and series total count for label_names labels that match the matchers.
// Messages are immediately sent as soon they reach message size threshold.
func labelValuesCardinality(
	lbNames []string,
	matchers []*labels.Matcher,
	idxReader tsdb.IndexReader,
	postingsForMatchersFn func(tsdb.IndexPostingsReader, ...*labels.Matcher) (index.Postings, error),
	msgSizeThreshold int,
	srv client.Ingester_LabelValuesCardinalityServer,
) error {
	ctx := srv.Context()

	resp := client.LabelValuesCardinalityResponse{}
	respSize := 0

	var mpc *index.PostingsCloner
	if len(matchers) > 0 {
		matchedPostings, err := postingsForMatchersFn(idxReader, matchers...)
		if err != nil {
			return err
		}
		mpc = index.NewPostingsCloner(matchedPostings)
	}

	for _, lbName := range lbNames {
		if err := ctx.Err(); err != nil {
			return err
		}
		allLblValues, err := idxReader.LabelValues(lbName)
		if err != nil {
			return err
		}
		// For each value count total number of series storing the result into cardinality response item.
		var respItem *client.LabelValueSeriesCount
		for _, lbValue := range allLblValues {
			if err := ctx.Err(); err != nil {
				return err
			}

			// Create label name response item entry.
			if respItem == nil {
				respItem = &client.LabelValueSeriesCount{
					LabelName:        lbName,
					LabelValueSeries: make(map[string]uint64),
				}
				resp.Items = append(resp.Items, respItem)
			}

			lbPostings, err := idxReader.Postings(lbName, lbValue)
			if err != nil {
				return err
			}

			if len(matchers) > 0 {
				lbPostings = index.Intersect(lbPostings, mpc.Clone())
			}

			seriesCount, err := postingsLength(ctx, lbPostings)
			if err != nil {
				return err
			}

			respItem.LabelValueSeries[lbValue] = seriesCount

			respSize += len(lbValue)
			if respSize < msgSizeThreshold {
				continue
			}
			// Flush the response when reached message threshold.
			if err := client.SendLabelValuesCardinalityResponse(srv, &resp); err != nil {
				return err
			}
			resp.Items = resp.Items[:0]
			respSize = 0
			respItem = nil
		}
	}
	// Send response in case there are any pending items.
	if len(resp.Items) > 0 {
		return client.SendLabelValuesCardinalityResponse(srv, &resp)
	}
	return nil
}

func postingsLength(ctx context.Context, p index.Postings) (uint64, error) {
	var l uint64
	for p.Next() {
		l++
		if l%checkContextErrorSeriesCount == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}
	}
	if p.Err() != nil {
		return 0, p.Err()
	}
	return l, nil
}
