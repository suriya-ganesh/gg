
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


# Running unique id tests
unique:
	$(call build_go_app,02_unique)

test_unique:
	./maelstrom/maelstrom test -w unique-ids --bin 02_unique/$(OUTPUT_BIN_DIR) --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition



# Running broadcast
broadcast_3a:
	$(call build_go_app,3a_single_broadcast)


test_broadcast_3a:
	./maelstrom/maelstrom test -w broadcast --bin 3a_single_broadcast/$(OUTPUT_BIN_DIR) --node-count 1 --time-limit 20 --rate 10



broadcast_3b:
	$(call build_go_app,3b_multi_node_broadcast)

test_broadcast_3b:
	./maelstrom/maelstrom test -w broadcast --bin 3b_multi_node_broadcast/$(OUTPUT_BIN_DIR) --node-count 5 --time-limit 20 --rate 10


broadcast_3c:
	$(call build_go_app,3c_networkp_broadcast)

test_broadcast_3c:
	./maelstrom/maelstrom test -w broadcast --bin 3c_networkp_broadcast/$(OUTPUT_BIN_DIR) --node-count 5 --time-limit 20 --rate 10 --nemesis partition


broadcast_3d:
	$(call build_go_app,3d_efficient_broadcast)

test_broadcast_3d:
	./maelstrom/maelstrom test -w broadcast --bin 3d_efficient_broadcast/$(OUTPUT_BIN_DIR) --node-count 25 --time-limit 20 --rate 10 --nemesis partition


# Clear all output directories
clear-all:
	find . -type d -name 'bin' -print0 | xargs -0 rm -rf:wq

serve:
	./maelstrom/maelstrom serve