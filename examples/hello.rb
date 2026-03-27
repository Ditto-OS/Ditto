# Ruby test script
puts "Hello from Ruby!"
puts "Running on Ditto embedded interpreter"

# Variables
name = "Ditto"
version = "0.1.0"

# String interpolation
puts "Project: #{name} v#{version}"

# Arithmetic
a = 10
b = 20
puts "Sum: #{a + b}"

# Array
colors = ["red", "green", "blue"]
puts "Colors: #{colors.length} items"

# Each loop
colors.each do |color|
  puts "  - #{color}"
end

# Times loop
3.times do
  puts "Counting..."
end

# If statement
x = 10
puts "x is big" if x > 5

# Method definition
def greet(person)
  puts "Hello, #{person}!"
end

greet("World")
