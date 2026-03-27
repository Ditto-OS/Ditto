#!/usr/bin/env python3
"""
Example Python script for testing Ditto
"""

def main():
    print("Hello from Python!")
    print(f"Running on Ditto polyglot runtime")
    
    # Show some Python features
    numbers = [1, 2, 3, 4, 5]
    squared = [x ** 2 for x in numbers]
    print(f"Squared numbers: {squared}")
    
    # Dictionary example
    data = {"name": "Ditto", "version": "0.1.0"}
    print(f"Project: {data['name']} v{data['version']}")

if __name__ == "__main__":
    main()
