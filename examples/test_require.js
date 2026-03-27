// Test JavaScript with require and async/await
const fs = require('fs');
const path = require('path');

console.log("fs module loaded");
console.log("path module loaded");

// Test path module
console.log(path.resolve('.'));

// Test async/await
async function fetchData() {
    console.log("Async function called!");
    console.log("Await completed!");
}

fetchData();

// Test console methods
console.info("Info message");
console.warn("Warning message");
console.error("Error message");
