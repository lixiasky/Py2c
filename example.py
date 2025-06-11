def add(x, y):
    return x + y

def greet(name):
    print("Hello,", name)

class Person:
    def __init__(self, name):
        self.name = name
        self.score = 100
    def say(self):
        print(self.name)
    def best_score(self):
        return self.score

greet("World")
a = 3
b = 4
c = add(a, b)
print(a, b, c)

p = Person("Tom")
p.say()
print("Best score:", p.best_score())

for i in range(5):
    if i == 2:
        continue
    if i == 4:
        break
    print(i)

if a > 1:
    print("a in range")
else:
    pass