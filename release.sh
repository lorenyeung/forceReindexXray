#!/bin/bash
LATEST_SCRIPT_TAG=$(curl -s https://api.github.com/repos/lorenyeung/forceReindexXray/releases/latest | jq -r '.tag_name')
LOCAL_TAG_NAME=$(jq -r '.script_version' metadata.json)
LOCAL_NAME=$(echo "${LOCAL_TAG_NAME/v/}")

if [ "$LATEST_SCRIPT_TAG" = "$LOCAL_TAG_NAME" ]; then
    echo "Did you forget to increment metadata.json? The latest release on github ($LATEST_SCRIPT_TAG) is the same as metadata.json's ($LOCAL_TAG_NAME)"
    select yn in "Yes" "No"; do
        case $yn in
            Yes)
                echo "Version please (no v):"
                read LOCAL_NAME
                LOCAL_TAG_NAME=v$LOCAL_NAME
                file=$(jq -r '.script_version="'$LOCAL_TAG_NAME'"' metadata.json)
                echo $file > metadata.json
                echo "double check:"
                cat metadata.json
                break;;
            No) echo "OK" ; break;;
        esac
    done
    else
        echo "Latest Github Tag:$LATEST_SCRIPT_TAG Local Tag:$LOCAL_TAG_NAME Local Name:$LOCAL_NAME"
fi
echo "Enter body description"
read message
body="{
  \"tag_name\": \"$LOCAL_TAG_NAME\",
  \"target_commitish\": \"master\",
  \"name\": \"$LOCAL_NAME\",
  \"body\": \"$message\",
  \"draft\": false,
  \"prerelease\": false
}"
object=$(git rev-parse HEAD)
echo "last commit is $object"
tag_body="{
  \"tag\": \"$LOCAL_TAG_NAME\",
  \"message\": \"$message\",
  \"object\": \"$object\",
  \"type\": \"commit\"
}"
echo "Body:"
echo "$body"
#echo "Tag Body:"
#echo $tag_body
echo "Looks good?"
    select yn in "Yes" "No"; do
        case $yn in
            Yes)
                echo $body > release.json 
                curl -u $GIT_USER:$GIT_TOKEN -XPOST https://api.github.com/repos/lorenyeung/forceReindexXray/releases -H "Content-Type: application/json" -T release.json -o release-response.json
                rm release.json
                BINARIES=("reindex-darwin-x64" "reindex-linux-x64")
                ASSET_URL=$(jq -r '.upload_url' release-response.json)
                edited=$(echo $ASSET_URL | sed 's/{?name,label}//')

                for BINARY in ${BINARIES[@]}; do
                    URL="$edited?name=$BINARY&label=$BINARY"
                    echo "uploading $BINARY to $URL"
                    curl -T $BINARY "$URL" -H "Content-type: application/x-binary" -u $GIT_USER:$GIT_TOKEN
                done
                rm release-response.json
                rm pkgdl-*
                break;;
            No) echo "OK" ; break;;
        esac
    done

