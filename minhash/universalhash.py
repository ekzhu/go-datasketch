'''
A experiment to show universal hashing produce
a uniformly distributed hashed values
'''
import matplotlib.pyplot as plt
import random

# Create a unversal hash function
def create_hashing(a, b, p, m):
    return lambda x : ((a*x+b) % p) % m

# Generate a random int32
random_int32 = lambda : random.random()*((1<<32)-1)

# Parameters for a universal hash function
a = random_int32()
b = random_int32()
# http://en.wikipedia.org/wiki/Mersenne_prime
p = (1<<61)-1
# The range of the function is [0, (1<<32)-1],
# or the 32-bit int
m = (1<<32)

h = create_hashing(a, b, p, m)
x = [h(random_int32()) for i in xrange(2**18)]

# See if we have a good range coverage
print (1<<32)-1, max(x), 0, min(x)

plt.hist(x, 100)
plt.savefig("universal_hash.png")
