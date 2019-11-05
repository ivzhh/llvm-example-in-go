// https://blog.felixangell.com/an-introduction-to-llvm-in-go

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ivzhh/go-llvm"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// setup our builder and module
	builder := llvm.NewBuilder()
	mod := llvm.NewModule("my_module")
	ctx := llvm.NewContext()

	AAA := llvm.AddGlobal(mod, llvm.Int32Type(), "AAA")

	members := []llvm.Type{
		llvm.Int32Type(),
		llvm.Int32Type(),
		llvm.Int32Type(),
		llvm.Int32Type(), // len
		llvm.Int32Type(), // cap
	}

	reqT := ctx.StructType(members, false)

	foo2T := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{llvm.Int32Type(), llvm.PointerType(llvm.Int32Type(), 0)}, false)
	foo2 := llvm.AddFunction(mod, "foo2", foo2T)

	{
		noinline := ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0)
		foo2.AddFunctionAttr(noinline)
		foo2.SetLinkage(llvm.ExternalLinkage)

		foo2.SetFunctionCallConv(llvm.GOSTACKCallConv)

		block := llvm.AddBasicBlock(foo2, "entry")
		builder.SetInsertPoint(block, block.FirstInstruction())

		p1 := builder.CreateLoad(foo2.Param(1), "param 1")

		builder.CreateStore(builder.CreateAdd(foo2.Param(0), p1, "add"), AAA)

		builder.CreateRet(llvm.ConstInt(llvm.Int32Type(), 0, false))
	}

	fooT := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{reqT, llvm.Int32Type(), llvm.Int32Type()}, false)
	foo := llvm.AddFunction(mod, "foo", fooT)

	{
		noinline := ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0)
		foo.AddFunctionAttr(noinline)
		foo.SetLinkage(llvm.ExternalLinkage)

		foo.SetFunctionCallConv(llvm.GOSTACKCallConv)

		block := llvm.AddBasicBlock(foo, "entry")
		builder.SetInsertPoint(block, block.FirstInstruction())

		p2 := builder.CreateAlloca(llvm.Int32Type(), "p2")

		builder.CreateStore(foo.Param(2), p2)

		result := builder.CreateCall(foo2, []llvm.Value{foo.Param(1), p2}, "foo2add")
		result.SetInstructionCallConv(llvm.GOSTACKCallConv)

		builder.CreateRet(builder.CreateAdd(foo.Param(1), builder.CreateLoad(mod.NamedGlobal("AAA"), "AAA"), "add"))

	}

	// create our function prologue
	main := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	llvm.AddFunction(mod, "main", main)
	block := llvm.AddBasicBlock(mod.NamedFunction("main"), "entry")
	builder.SetInsertPoint(block, block.FirstInstruction())

	// int a = 32
	a := builder.CreateAlloca(llvm.Int32Type(), "a")
	builder.CreateStore(llvm.ConstInt(llvm.Int32Type(), 32, false), a)

	// int b = 16
	b := builder.CreateAlloca(llvm.Int32Type(), "b")
	builder.CreateStore(llvm.ConstInt(llvm.Int32Type(), 16, false), b)

	req := builder.CreateAlloca(reqT, "req")

	// return a + b
	bVal := builder.CreateLoad(b, "b_val")
	aVal := builder.CreateLoad(a, "a_val")

	reqVal := builder.CreateLoad(req, "req")
	reqVal = builder.CreateInsertValue(reqVal, llvm.ConstInt(llvm.Int32Type(), 0, false), 0, "set up value")
	reqVal = builder.CreateInsertValue(reqVal, llvm.ConstInt(llvm.Int32Type(), 1, false), 1, "set up value")
	reqVal = builder.CreateInsertValue(reqVal, llvm.ConstInt(llvm.Int32Type(), 2, false), 2, "set up value")
	reqVal = builder.CreateInsertValue(reqVal, llvm.ConstInt(llvm.Int32Type(), 3, false), 3, "set up value")
	reqVal = builder.CreateInsertValue(reqVal, llvm.ConstInt(llvm.Int32Type(), 4, false), 4, "set up value")

	result := builder.CreateCall(foo, []llvm.Value{reqVal, aVal, bVal}, "ab_val")
	result.SetInstructionCallConv(llvm.GOSTACKCallConv)
	builder.CreateRet(result)

	// verify it's all good
	if ok := llvm.VerifyModule(mod, llvm.ReturnStatusAction); ok != nil {
		fmt.Println(ok.Error())
	}
	ioutil.WriteFile("a.ll", []byte(mod.String()), 0644)

	if err := llvm.InitializeNativeTarget(); err != nil {
		log.Fatalf("error: %+v", err)
		os.Exit(-1)
	}

	//triple := llvm.DefaultTargetTriple()

	triple := "x86_64-pc-linux-gnu"

	log.Printf("triple: %s", triple)

	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		log.Fatalf("error: %+v", err)
		os.Exit(-1)
	}

	// this section is need to create target and print asm
	// https://llvm.org/docs/tutorial/MyFirstLanguageFrontend/LangImpl08.html
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()

	machine := target.CreateTargetMachine(triple, "", "", llvm.CodeGenLevelDefault, llvm.RelocStatic, llvm.CodeModelDefault)

	llvmBuf, err := machine.EmitToMemoryBuffer(mod, llvm.AssemblyFile)
	if err != nil {
		log.Fatalf("error: %+v", err)
		os.Exit(-1)
	}
	ioutil.WriteFile("a.S", llvmBuf.Bytes(), 0644)

	// create our exe engine
	engine, err := llvm.NewMCJITCompiler(mod, llvm.NewMCJITCompilerOptions())
	if err != nil {
		fmt.Println(err.Error())
	}

	// run the function!
	funcResult := engine.RunFunction(mod.NamedFunction("main"), []llvm.GenericValue{})
	fmt.Printf("%d\n", funcResult.Int(false))
}
