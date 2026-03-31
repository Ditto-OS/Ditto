// Test script for embedded lodash package
const _ = require('lodash');

console.log("Testing lodash module...");

// Test chunk function
const arr = [1, 2, 3, 4, 5];
const chunked = _.chunk(arr, 2);
console.log("Chunked:", JSON.stringify(chunked));

// Test compact
const withEmpty = [1, false, 2, null, 3];
const compacted = _.compact(withEmpty);
console.log("Compacted:", JSON.stringify(compacted));

console.log("Lodash loaded successfully!");
