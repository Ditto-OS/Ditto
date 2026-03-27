-- Example Lua script for testing Ditto
print("Hello from Lua!")
print("Running on Ditto embedded interpreter")

-- Variables
local name = "Ditto"
local version = 0.1

-- String concatenation
io.write("Project: " .. name .. " v" .. version .. "\n")

-- Numbers
local a = 10
local b = 20
print("Sum: " .. (a + b))

-- For loop
print("Counting:")
for i = 1, 5 do
    print("  " .. i)
end

-- Table
local colors = {"red", "green", "blue"}
print("Colors: " .. #colors .. " items")

-- Functions
function greet(person)
    print("Hello, " .. person .. "!")
end

greet("World")
