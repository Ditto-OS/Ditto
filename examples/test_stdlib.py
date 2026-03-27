# Test Python with standard library and classes
import math
import os

# Test math module
print("Math module tests:")
print(math.sqrt(16))
print(math.pi)

# Test os module
print("OS module tests:")
print(os.getcwd())
print(os.name)

# Test class
print("Class test:")
class Person:
    def __init__(self, name):
        self.name = name

p = Person()
print("Created Person instance")
