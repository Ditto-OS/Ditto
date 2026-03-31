# Test script for embedded packages
import requests

print("Testing requests module...")
print("Module loaded successfully!")

# Try to use it
resp = requests.get("http://example.com")
print(resp.text)

print("Done!")
