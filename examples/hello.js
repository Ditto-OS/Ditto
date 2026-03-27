/**
 * Example Node.js script for testing Ditto
 */

function main() {
    console.log("Hello from Node.js!");
    console.log("Running on Ditto polyglot runtime");
    
    // Show some JavaScript features
    const numbers = [1, 2, 3, 4, 5];
    const squared = numbers.map(x => x ** 2);
    console.log(`Squared numbers: ${squared}`);
    
    // Object example
    const project = { name: "Ditto", version: "0.1.0" };
    console.log(`Project: ${project.name} v${project.version}`);
}

main();
