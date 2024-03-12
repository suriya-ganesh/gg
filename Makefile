
OUTPUT_BIN_DIR = bin/build

# This function builds the Go application in the specified directory
define build_go_app
	$(info Building Go application in directory: $(1))
	cd $(1) && go build -o $(OUTPUT_BIN_DIR)
endef


# Running echo tests
echo:
	$(call build_go_app,01_echo)

test_echo:
	./maelstrom/maelstrom test -w echo --bin 01_echo/$(OUTPUT_BIN_DIR) --node-count 1 --time-limit 10


# Running unique id tests;q
unique:
	$(call build_go_app,02_unique)

test_unique:
	./maelstrom/maelstrom test -w unique-ids --bin 02_unique/$(OUTPUT_BIN_DIR) --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

# Clear all output directories
clear-all:
	find . -type d -name 'bin' -print0 | xargs -0 rm -rf:wq