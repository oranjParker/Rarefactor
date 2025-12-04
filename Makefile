.PHONY: gen backend-run

# 1. Generate ALL code (Python + TS) from the root protos
gen:
	mkdir -p backend/generated
	# mkdir -p frontend/src/generated
	buf generate protos

backend-run:
	cd backend && uv run uvicorn main:app --reload --port 8000

crawl:
	cd backend && uv run start_crawl.py