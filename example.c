#include <stdio.h>

typedef struct {
    char* name;
    double score;
} Person;
void Person___init__(Person* self, char* name) {
        self->name = name;
        self->score = 100;
}
void Person_say(Person* self) {
        printf("%s\n", self->name);

}
double Person_best_score(Person* self) {
        return self->score;
}
    void add(double x, double y, double* result) {
        *result = (x + y);
    }
    void greet(char* name) {
        printf("%s %s\n", "Hello,", name);

    }
int main() {
greet("World");
    double a = 3;
    double b = 4;
    double c;
    add(a, b, &c);
    printf("%f %f %f\n", a, b, c);
;
    Person p;
    Person___init__(&p, "Tom");
Person_say(&p);
    printf("%s %s\n", "Best score:", Person_best_score(&p));
;
    for (int i = 0; i < 5; i++) {
        if (i == 2) {
            continue;
        }
        if (i == 4) {
            break;
        }
        printf("%f\n", i);

    }
    if (a > 1) {
        printf("%s\n", "a in range");

    }
    else {
        // pass
    }
    return 0;
}
