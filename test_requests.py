# Test Python with embedded requests package
import requests

# Test requests module
print("Testing requests module:")
try:
    response = requests.get("https://api.example.com/data")
    print("Response status:", response.status_code)
    print("Response text:", response.text)
except Exception as e:
    print("Error:", e)