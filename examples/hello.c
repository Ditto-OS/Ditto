// Example C script for testing Ditto
#include <stdio.h>

int main(int argc, char *argv[]) {
    // Basic output
    printf("Hello from C!\n");
    printf("Running on Ditto embedded interpreter\n\n");
    
    // Variables
    int number = 42;
    float pi = 3.14159;
    char *message = "Ditto is awesome";
    
    printf("Number: %d\n", number);
    printf("Pi: %f\n", pi);
    printf("Message: %s\n\n", message);
    
    // Arithmetic
    int a = 10, b = 5;
    printf("Arithmetic:\n");
    printf("  %d + %d = %d\n", a, b, a + b);
    printf("  %d - %d = %d\n", a, b, a - b);
    printf("  %d * %d = %d\n", a, b, a * b);
    printf("  %d / %d = %d\n", a, b, a / b);
    
    // Loop
    printf("\nCounting: ");
    for (int i = 1; i <= 5; i++) {
        printf("%d ", i);
    }
    printf("\n");
    
    return 0;
}
