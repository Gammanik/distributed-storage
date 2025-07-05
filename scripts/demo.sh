#!/bin/bash
# scripts/demo.sh

set -e

echo "==== Checking prerequisites ===="
# Проверка доступности REST сервера
until $(curl --output /dev/null --silent --head --fail http://localhost:8080); do
    printf '.'
    sleep 1
done
echo "REST server is up and running!"

echo "==== Generating test file ===="
dd if=/dev/urandom of=testfile.bin bs=1M count=10

echo "==== Calculating original hash ===="
# Для macOS используем shasum вместо sha256sum
if command -v sha256sum > /dev/null; then
  ORIGINAL_HASH=$(sha256sum testfile.bin | awk '{print $1}')
else
  ORIGINAL_HASH=$(shasum -a 256 testfile.bin | awk '{print $1}')
fi
echo "Original file hash: $ORIGINAL_HASH"

echo "==== Uploading file ===="
FILE_ID=$(curl -s -X POST -H "X-Filename: testfile.bin" --data-binary @testfile.bin http://localhost:8080/upload)
echo "Uploaded file ID: $FILE_ID"

echo "==== Downloading file ===="
curl -s -o downloaded.bin "http://localhost:8080/download?fileID=$FILE_ID"

echo "==== Calculating downloaded hash ===="
# Для macOS используем shasum вместо sha256sum
if command -v sha256sum > /dev/null; then
  DOWNLOADED_HASH=$(sha256sum downloaded.bin | awk '{print $1}')
else
  DOWNLOADED_HASH=$(shasum -a 256 downloaded.bin | awk '{print $1}')
fi
echo "Downloaded file hash: $DOWNLOADED_HASH"

echo "==== Verifying integrity ===="
if [ "$ORIGINAL_HASH" = "$DOWNLOADED_HASH" ]; then
    echo "✅ SUCCESS: Files are identical."
else
    echo "❌ FAILURE: Files are different."
fi

# Очистка
echo "==== Cleaning up ===="
rm testfile.bin downloaded.bin