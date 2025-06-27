package structural

type PackageBuilder struct {
	err error
	p   AnalyzedPackage
}

func MakePackageBuilder() PackageBuilder {
	return PackageBuilder{}
}

func (b PackageBuilder) Package() AnalyzedPackage {
	if b.err == nil {
		return b.p
	}

	return AnalyzedPackage{}
}
