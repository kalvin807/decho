BINARY_NAME=decho

all: build

build:
	go build -o $(BINARY_NAME) -v

clean:
	go clean
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)
