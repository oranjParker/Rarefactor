.PHONY: gen lint run

# 1. Lint your protos (Senior Engineer move)
lint:
	buf lint protos

# 2. Generate code using the configuration in buf.gen.yaml
gen:
	buf generate protos

# 3. Run the server
run:
	uv run server.py