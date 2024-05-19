build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc  CXX=x86_64-linux-musl-g++ go build -ldflags="-s -w -extldflags=-static" -o 5028-week3

deploy:
	ssh CuBoulder-DO "rm /root/5028-week3"; \
	scp ./5028-week3 CuBoulder-DO:/root/5028-week3 && \
	ssh CuBoulder-DO "systemctl restart 5028-week3"