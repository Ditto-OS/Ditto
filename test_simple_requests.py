# Test Python with embedded requests package
import requests

# Test that the module loads
print("Requests module loaded successfully!")
print("Version:", requests.__version__)
print("Author:", requests.__author__)

# Test a simple GET request
response = requests.get("https://api.example.com/data")
print("Status code:", response.status_code)
print("Response text:", response.text)