package sqrt_test

import (
	"fmt"

	"github.com/keep94/sqrt"
)

func ExampleContext_Sqrt() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// Print the square root of 13 with 100 significant digits.
	fmt.Printf("%.100g\n", ctxt.Sqrt(13))
	// Output:
	// 3.605551275463989293119221267470495946251296573845246212710453056227166948293010445204619082018490717
}

func ExampleContext_CubeRoot() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// Print the cube root of 3 with 100 significant digits.
	fmt.Printf("%.100g\n", ctxt.CubeRoot(3))
	// Output:
	// 1.442249570307408382321638310780109588391869253499350577546416194541687596829997339854755479705645256
}

func ExampleContext_NewNumberForTesting() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// n = 10.2003400340034...
	n, _ := ctxt.NewNumberForTesting([]int{1, 0, 2}, []int{0, 0, 3, 4}, 2)

	fmt.Println(n)
	// Output:
	// 10.20034003400340
}

func ExampleContext_NewFiniteNumber() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// n = 563.5
	n, _ := ctxt.NewFiniteNumber([]int{5, 6, 3, 5}, 3)

	fmt.Println(n.Exact())
	// Output:
	// 563.5
}

func ExampleAsString() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// sqrt(3) = 0.1732050807... * 10^1
	n := ctxt.Sqrt(3)

	fmt.Println(sqrt.AsString(n.WithStart(2).WithEnd(10)))
	// Output:
	// 32050807
}

func ExampleFiniteNumber_Exponent() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// sqrt(50176) = 0.224 * 10^3
	n := ctxt.Sqrt(50176)

	fmt.Println(n.Exponent())
	// Output:
	// 3
}

func ExampleFiniteNumber_All() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// sqrt(7) = 0.26457513110... * 10^1
	n := ctxt.Sqrt(7)

	for index, value := range n.All() {
		fmt.Println(index, value)
		if index == 5 {
			break
		}
	}
	// Output:
	// 0 2
	// 1 6
	// 2 4
	// 3 5
	// 4 7
	// 5 5
}

func ExampleFiniteNumber_Values() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// sqrt(7) = 0.26457513110... * 10^1
	n := ctxt.Sqrt(7)

	for value := range n.WithEnd(6).Values() {
		fmt.Println(value)
	}
	// Output:
	// 2
	// 6
	// 4
	// 5
	// 7
	// 5
}

func ExampleFiniteNumber_Backward() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// sqrt(7) = 0.26457513110... * 10^1
	n := ctxt.Sqrt(7).WithSignificant(6)

	for index, value := range n.Backward() {
		fmt.Println(index, value)
	}
	// Output:
	// 5 5
	// 4 7
	// 3 5
	// 2 4
	// 1 6
	// 0 2
}

func ExampleFiniteNumber_Exact() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	n := ctxt.Sqrt(2).WithSignificant(60)
	fmt.Println(n.Exact())
	// Output:
	// 1.41421356237309504880168872420969807856967187537694807317667
}

func ExampleFiniteNumber_At() {
	var ctxt sqrt.Context
	defer ctxt.Close()

	// sqrt(7) = 0.264575131106459...*10^1
	n := ctxt.Sqrt(7)

	fmt.Println(n.At(0))
	fmt.Println(n.At(1))
	fmt.Println(n.At(2))
	// Output:
	// 2
	// 6
	// 4
}
