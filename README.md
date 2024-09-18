ngrock for files


echo README.md | xargs -I{} curl -v -X POST -F "file=@{}" http://localhost:8080/upload/123456789012345678901234567890
