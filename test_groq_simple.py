# Test Python with groq package - simple import test
import groq

# Test that the module loads
print("Groq module loaded successfully!")
print("Groq module:", groq)

# Check if Groq class exists
if hasattr(groq, 'Groq'):
    print("Groq class found in module!")
else:
    print("Groq class not found")

# List some attributes
print("Module attributes:", dir(groq)[:10])  # Show first 10 attributes