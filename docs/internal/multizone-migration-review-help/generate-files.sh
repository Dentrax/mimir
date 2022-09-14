#!/bin/env bash

python3 extract-yamls.py ../../sources/migration-guide/migrating-from-single-zone-with-helm.md

SED="sed"

for component in alertmanager ingester storegateway ; do
  pushd "${component}"
  rm -rf step0
  helm template krajo ../../../../operations/helm/charts/mimir-distributed --output-dir step0 -f ../base.yaml
  find step0 -type f -print0 | xargs -0 "${SED}" -E -i -- "/^\s+(checksum\/config|(helm.sh\/)?chart|app.kubernetes.io\/version|image: \"grafana\/(mimir|mimir-continuous-test|enterprise-metrics)):/d"
  i=1
  while [ -e "${component}-step${i}.yaml" ] ; do
    step_yaml="${component}-step${i}.yaml"
    step_dir="step${i}"
    prev=`expr $i - 1`
    prev_dir="step$(expr $i - 1)"
    echo "Component=${component} Prev step=${prev_dir} Current step=${step_dir}"
    rm -rf "${step_dir}"
    ${SED} -i "s/<N>/3/g" "${step_yaml}"
    helm template krajo ../../../../operations/helm/charts/mimir-distributed --output-dir "${step_dir}" -f ../base.yaml -f "${step_yaml}"
    find "${step_dir}" -type f -print0 | xargs -0 "${SED}" -E -i -- "/^\s+(checksum\/config|(helm.sh\/)?chart|app.kubernetes.io\/version|image: \"grafana\/(mimir|mimir-continuous-test|enterprise-metrics)):/d"
    diff -c -r "${prev_dir}" "${step_dir}" > "diff-${prev_dir}-${step_dir}.patch"
    ((i++))
  done
  popd
done
