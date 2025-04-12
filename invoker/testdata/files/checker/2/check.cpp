#include "testlib.h"

using namespace std;

int main(int argc, char *argv[]) {
    registerTestlibCmd(argc, argv);

    int a, b;
    a = ouf.readInt();
    b = ans.readInt();
    quitp(a + b + 1, "Points");
}