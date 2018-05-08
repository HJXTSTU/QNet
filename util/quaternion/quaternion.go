package quaternion

type Quaternion interface{
	Mul(Quaternion)Quaternion
	Conjugate()Quaternion
	W()float32
	X()float32
	Y()float32
	Z()float32
}

func Identity()Quaternion{
	return quaternion{1,0,0,0}
}

func NewQuaternion(w,x,y,z float32)Quaternion{
	return quaternion{w,x,y,z}
}

type quaternion struct{
	w float32
	x float32
	y float32
	z float32
}

func (this quaternion)X()float32{
	return this.x
}

func (this quaternion)Y()float32{
	return this.y
}

func (this quaternion)Z()float32{
	return this.z
}

func (this quaternion)W()float32{
	return this.w
}

func (this quaternion)Conjugate()Quaternion{
	res := quaternion{this.w,-this.x,-this.y,-this.z}
	return res
}

func (left quaternion)Mul(right Quaternion)Quaternion{
	nw := left.W()*right.W()-left.X()*right.X()-left.Y()*right.Y()-left.Z()*right.Z()
	nx := left.W()*right.X()+left.X()*right.W()+left.Y()*right.Z()-left.Z()*right.Y()
	ny := left.W()*right.Y()-left.X()*right.Z()+left.Y()*right.W()+left.Z()*right.X()
	nz := left.W()*right.Z()+left.X()*right.Y()-left.Y()*right.X()+left.Z()*right.W()
	res := quaternion{
		nw,
		nx,
		ny,
		nz,
	}
	return res
}


