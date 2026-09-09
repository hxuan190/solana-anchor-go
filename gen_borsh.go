package main

import (
	. "github.com/dave/jennifer/jen"
)

// genBorshFile builds "borsh.go": local replacements for the pieces of gagliardetto/binary that
// fluxrpc/solana-go/binary has no equivalent for — an 8-byte discriminator type, 128-bit integer
// types (raw little-endian byte layout, which is also the correct Borsh wire format), and
// float32/64 helpers (fluxrpc's Encoder/Decoder have no float support at all).
func genBorshFile(idl IDL) (*FileWrapper, error) {
	file := NewGoFile(idl.Metadata.Name, true)

	file.Add(
		Comment("TypeID is an 8-byte Anchor/Borsh discriminator.").Line(),
		Type().Id("TypeID").Index(Lit(8)).Byte().Line(),
	)
	file.Add(
		Func().Params(Id("id").Id("TypeID")).Id("Bytes").
			Params().
			Params(Index().Byte()).
			Block(Return(Id("id").Index(Op(":")))).
			Line(),
	)

	file.Add(
		Comment("Uint128 is the raw little-endian byte layout of a Borsh u128.").Line(),
		Type().Id("Uint128").Index(Lit(16)).Byte().Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Uint128")).Id("MarshalWithEncoder").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Id("encoder").Dot("WriteBytes").Call(Id("v").Index(Op(":")))
				body.Return(Id("encoder").Dot("Err").Call())
			}).Line(),
	)
	file.Add(
		Func().Params(Id("v").Op("*").Id("Uint128")).Id("UnmarshalWithDecoder").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Copy(Id("v").Index(Op(":")), Id("decoder").Dot("ReadBytes").Call(Lit(16)))
				body.Return(Id("decoder").Dot("Err").Call())
			}).Line(),
	)

	file.Add(
		Comment("BigInt reads the little-endian bytes as an unsigned 128-bit integer.").Line(),
		Func().Params(Id("v").Id("Uint128")).Id("BigInt").
			Params().
			Params(Op("*").Qual("math/big", "Int")).
			BlockFunc(func(body *Group) {
				body.Var().Id("be").Index(Lit(16)).Byte()
				body.For(Id("i").Op(":=").Lit(0), Id("i").Op("<").Lit(16), Id("i").Op("++")).Block(
					Id("be").Index(Id("i")).Op("=").Id("v").Index(Lit(15).Op("-").Id("i")),
				)
				body.Return(Qual("math/big", "NewInt").Call(Lit(0)).Dot("SetBytes").Call(Id("be").Index(Op(":"))))
			}).Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Uint128")).Id("String").
			Params().
			Params(String()).
			Block(Return(Id("v").Dot("BigInt").Call().Dot("String").Call())).
			Line(),
	)
	file.Add(
		Comment("MarshalJSON emits the decimal value as a JSON string, matching how").Line(),
		Comment("gagliardetto/binary serialised u128 fields.").Line(),
		Func().Params(Id("v").Id("Uint128")).Id("MarshalJSON").
			Params().
			Params(Index().Byte(), Error()).
			Block(Return(Index().Byte().Call(Lit(`"`).Op("+").Id("v").Dot("String").Call().Op("+").Lit(`"`)), Nil())).
			Line(),
	)
	file.Add(
		Func().Params(Id("v").Op("*").Id("Uint128")).Id("UnmarshalJSON").
			Params(Id("data").Index().Byte()).
			Params(Error()).
			BlockFunc(func(body *Group) {
				body.List(Id("parsed"), Id("null"), Err()).Op(":=").Id("parse128").Call(Id("data"))
				body.If(Err().Op("!=").Nil().Op("||").Id("null")).Block(Return(Err()))
				body.If(Id("parsed").Dot("Sign").Call().Op("<").Lit(0).Op("||").Id("parsed").Dot("BitLen").Call().Op(">").Lit(128)).Block(
					Return(Qual("fmt", "Errorf").Call(Lit("u128 out of range: %s"), Id("parsed"))),
				)
				body.Op("*").Id("v").Op("=").Id("Uint128").Call(Id("bytes128").Call(Id("parsed")))
				body.Return(Nil())
			}).Line(),
	)

	file.Add(
		Comment("Int128 is byte-identical to Uint128 — two's-complement is already correct as raw little-endian bytes.").Line(),
		Type().Id("Int128").Index(Lit(16)).Byte().Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Int128")).Id("MarshalWithEncoder").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Id("encoder").Dot("WriteBytes").Call(Id("v").Index(Op(":")))
				body.Return(Id("encoder").Dot("Err").Call())
			}).Line(),
	)
	file.Add(
		Func().Params(Id("v").Op("*").Id("Int128")).Id("UnmarshalWithDecoder").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Copy(Id("v").Index(Op(":")), Id("decoder").Dot("ReadBytes").Call(Lit(16)))
				body.Return(Id("decoder").Dot("Err").Call())
			}).Line(),
	)

	file.Add(
		Comment("BigInt reads the little-endian bytes as a signed two's-complement 128-bit integer.").Line(),
		Func().Params(Id("v").Id("Int128")).Id("BigInt").
			Params().
			Params(Op("*").Qual("math/big", "Int")).
			BlockFunc(func(body *Group) {
				body.Id("out").Op(":=").Id("Uint128").Call(Id("v")).Dot("BigInt").Call()
				body.If(Id("v").Index(Lit(15)).Op("&").Lit(0x80).Op("!=").Lit(0)).Block(
					Id("out").Dot("Sub").Call(Id("out"), Qual("math/big", "NewInt").Call(Lit(0)).Dot("Lsh").Call(Qual("math/big", "NewInt").Call(Lit(1)), Lit(128))),
				)
				body.Return(Id("out"))
			}).Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Int128")).Id("String").
			Params().
			Params(String()).
			Block(Return(Id("v").Dot("BigInt").Call().Dot("String").Call())).
			Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Int128")).Id("MarshalJSON").
			Params().
			Params(Index().Byte(), Error()).
			Block(Return(Index().Byte().Call(Lit(`"`).Op("+").Id("v").Dot("String").Call().Op("+").Lit(`"`)), Nil())).
			Line(),
	)
	file.Add(
		Func().Params(Id("v").Op("*").Id("Int128")).Id("UnmarshalJSON").
			Params(Id("data").Index().Byte()).
			Params(Error()).
			BlockFunc(func(body *Group) {
				body.List(Id("parsed"), Id("null"), Err()).Op(":=").Id("parse128").Call(Id("data"))
				body.If(Err().Op("!=").Nil().Op("||").Id("null")).Block(Return(Err()))
				body.Id("max").Op(":=").Qual("math/big", "NewInt").Call(Lit(0)).Dot("Sub").Call(Qual("math/big", "NewInt").Call(Lit(0)).Dot("Lsh").Call(Qual("math/big", "NewInt").Call(Lit(1)), Lit(127)), Qual("math/big", "NewInt").Call(Lit(1)))
				body.Id("min").Op(":=").Qual("math/big", "NewInt").Call(Lit(0)).Dot("Neg").Call(Qual("math/big", "NewInt").Call(Lit(0)).Dot("Lsh").Call(Qual("math/big", "NewInt").Call(Lit(1)), Lit(127)))
				body.If(Id("parsed").Dot("Cmp").Call(Id("min")).Op("<").Lit(0).Op("||").Id("parsed").Dot("Cmp").Call(Id("max")).Op(">").Lit(0)).Block(
					Return(Qual("fmt", "Errorf").Call(Lit("i128 out of range: %s"), Id("parsed"))),
				)
				body.If(Id("parsed").Dot("Sign").Call().Op("<").Lit(0)).Block(
					Id("parsed").Op("=").Qual("math/big", "NewInt").Call(Lit(0)).Dot("Add").Call(Id("parsed"), Qual("math/big", "NewInt").Call(Lit(0)).Dot("Lsh").Call(Qual("math/big", "NewInt").Call(Lit(1)), Lit(128))),
				)
				body.Op("*").Id("v").Op("=").Id("Int128").Call(Id("bytes128").Call(Id("parsed")))
				body.Return(Nil())
			}).Line(),
	)
	file.Add(
		Comment("parse128 accepts a quoted or bare JSON number; null leaves the value untouched.").Line(),
		Func().Id("parse128").
			Params(Id("data").Index().Byte()).
			Params(Op("*").Qual("math/big", "Int"), Bool(), Error()).
			BlockFunc(func(body *Group) {
				body.Id("s").Op(":=").String().Call(Id("data"))
				body.If(Id("s").Op("==").Lit("null")).Block(Return(Nil(), True(), Nil()))
				body.If(Len(Id("s")).Op(">=").Lit(2).Op("&&").Id("s").Index(Lit(0)).Op("==").LitRune('"').Op("&&").Id("s").Index(Len(Id("s")).Op("-").Lit(1)).Op("==").LitRune('"')).Block(
					Id("s").Op("=").Id("s").Index(Lit(1), Len(Id("s")).Op("-").Lit(1)),
				)
				body.List(Id("parsed"), Id("ok")).Op(":=").Qual("math/big", "NewInt").Call(Lit(0)).Dot("SetString").Call(Id("s"), Lit(0))
				body.If(Op("!").Id("ok")).Block(
					Return(Nil(), False(), Qual("fmt", "Errorf").Call(Lit("cannot parse 128-bit integer %q"), Id("s"))),
				)
				body.Return(Id("parsed"), False(), Nil())
			}).Line(),
	)
	file.Add(
		Comment("bytes128 renders a non-negative big.Int as 16 little-endian bytes.").Line(),
		Func().Id("bytes128").
			Params(Id("v").Op("*").Qual("math/big", "Int")).
			Params(Index(Lit(16)).Byte()).
			BlockFunc(func(body *Group) {
				body.Var().Id("be").Index(Lit(16)).Byte()
				body.Id("v").Dot("FillBytes").Call(Id("be").Index(Op(":")))
				body.Var().Id("le").Index(Lit(16)).Byte()
				body.For(Id("i").Op(":=").Lit(0), Id("i").Op("<").Lit(16), Id("i").Op("++")).Block(
					Id("le").Index(Id("i")).Op("=").Id("be").Index(Lit(15).Op("-").Id("i")),
				)
				body.Return(Id("le"))
			}).Line(),
	)

	// fluxrpc/solana-go/binary has no float32/64 support at all: encode/decode via IEEE-754 bit
	// patterns over the existing uint32/64 primitives.
	file.Add(
		Func().Id("WriteFloat32").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder"), Id("v").Float32()).
			Block(
				Id("encoder").Dot("WriteUint32").Call(Qual("math", "Float32bits").Call(Id("v"))),
			).Line(),
	)
	file.Add(
		Func().Id("ReadFloat32").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Float32()).
			Block(
				Return(Qual("math", "Float32frombits").Call(Id("decoder").Dot("ReadUint32").Call())),
			).Line(),
	)
	file.Add(
		Func().Id("WriteFloat64").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder"), Id("v").Float64()).
			Block(
				Id("encoder").Dot("WriteUint64").Call(Qual("math", "Float64bits").Call(Id("v"))),
			).Line(),
	)
	file.Add(
		Func().Id("ReadFloat64").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Float64()).
			Block(
				Return(Qual("math", "Float64frombits").Call(Id("decoder").Dot("ReadUint64").Call())),
			).Line(),
	)

	return &FileWrapper{
		Name: "borsh",
		File: file,
	}, nil
}
