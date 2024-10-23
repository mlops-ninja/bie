ngrock for files


echo README.md | xargs -I{} curl -v -X POST -F "file=@{}" http://localhost:8080/upload/vQbSGLoBuvWVL5539lkadH80GoHGISmG
