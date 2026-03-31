// Test JavaScript with embedded lodash package
const _ = require('lodash');

// Test that the module loads
console.log("Lodash module loaded successfully!");

// Test some lodash functions
const arr = [1, 2, 3, 4, 5];
const chunked = _.chunk(arr, 2);
console.log("Chunked array:", chunked);

const reversed = _.reverse(arr);
console.log("Reversed array:", reversed);