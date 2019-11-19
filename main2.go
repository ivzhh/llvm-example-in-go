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
	//ctx := llvm.NewContext()

	// create our function prologue
	i32x4 := llvm.VectorType(llvm.Int32Type(), 4)
	mainT := llvm.FunctionType(i32x4, []llvm.Type{i32x4}, false)
	main := llvm.AddFunction(mod, "main", mainT)
	block := llvm.AddBasicBlock(mod.NamedFunction("main"), "entry")
	builder.SetInsertPoint(block, block.FirstInstruction())

	v := builder.CreateInsertElement(main.Param(0), llvm.ConstInt(llvm.Int32Type(), 12, false), llvm.ConstInt(llvm.Int32Type(), 1, false), "insert")

	builder.CreateRet(v)

	// verify it's all good
	if ok := llvm.VerifyModule(mod, llvm.ReturnStatusAction); ok != nil {
		fmt.Println(ok.Error())
	}
	ioutil.WriteFile("b.ll", []byte(mod.String()), 0644)

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
	ioutil.WriteFile("b.S", llvmBuf.Bytes(), 0644)
}
