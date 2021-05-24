BIN=bin

local: 
	mkdir $(BIN) || echo "Exists"
	(cd geck; go build -o ../$(BIN)/geck)
	(cd frontend && make pack && cp garden-webdata.tar.gz ../$(BIN)/ )

