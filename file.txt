ngrock for files


echo README.md | xargs -I{} curl -v -X POST -F "file=@{}" https://bie.mlops.ninja/upload/SNd594uNdqlVIVz06twS9FM674iUFFFY
