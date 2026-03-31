# Test Python with groq package
from groq import Groq

# Test that the module loads
print("Groq module loaded successfully!")

# Initialize the client (this won't actually work without API key, but tests loading)
try:
    client = Groq()
    print("Groq client created successfully!")
    print("Client type:", type(client))
except Exception as e:
    print("Error creating client:", e)