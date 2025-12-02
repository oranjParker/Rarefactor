from typing import Any, List
from google.protobuf.field_mask_pb2 import FieldMask

def apply_field_mask(source: Any, target: Any, mask: FieldMask) -> None:
    for path in mask.paths:
        if not hasattr(source, path) or not hasattr(target, path):
            continue

        value = getattr(source, path)

        setattr(target, path, value)