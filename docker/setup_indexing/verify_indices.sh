#!/usr/bin/env sh

# Verify that the indices are created and index templates are defined.
# The first argument is the URL to the cluster. It must have a HTTP(s) scheme.

if [[ -z "$1" ]] ; then
    echo 'ES URL is missing'
    exit 1
fi

es_cluster_url=$1

check_index_template_existence()
{
  template=$1
  res=`curl -X GET -s "$es_cluster_url/_template/${template}" -H "Content-type: application/json"`
  if [[ $res == "{}" ]]; then
    echo -e "\033[1;31m  [Not OK] \033[0m Template $template does not exist !"
  else
    version=`echo $res | jq .${template}.version`
    echo -e "\033[1;32m  [OK] \033[0m Template $template exists!"
  fi
}

echo "Verifying index template existence..."
index_template_name=flight
check_index_template_existence $index_template_name

