import sys
print("Python version:", sys.version)

import os
print("Current dir:", os.getcwd())

import math
print("sqrt(16):", math.sqrt(16))

# Test with requests package
try:
    import requests
    print("requests module loaded!")
    resp = requests.get("http://example.com")
    print("Response:", resp.text)
except Exception as e:
    print("requests error:", e)
