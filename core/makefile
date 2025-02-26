clash:clear
	cd ./lib && go build -o libclash.so -buildmode=c-shared -ldflags='-w -s'  bridge.go

clear:
	cd ./lib && rm -rf libclash*
