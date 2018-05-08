package vector

import (
	. "wwt/util/quaternion"
	"math"
)

type Vector3 interface{
	Rotate(quaternion Quaternion)Vector3
	Dot(Vector3)float32
	Cross(Vector3)Vector3
	Manitude()float32
	SqrtManitude()float32
	Normalize()Vector3
	Mul(float32)Vector3
	Div(float32)Vector3
	Add(Vector3)Vector3
	Sub(v Vector3)Vector3
	X()float32
	Y()float32
	Z()float32
}

func Zero()Vector3{
	return vector3{0,0,0}
}

func Forward()Vector3{
	return vector3{0,0,1}
}

func Up()Vector3{
	return vector3{0,1,0}
}

func Right()Vector3{
	return vector3{1,0,0}
}

func NewVector3(x,y,z float32)Vector3{
	return vector3{x,y,z}
}

type vector3 struct{
	x float32
	y float32
	z float32
}

func (s vector3)Sub(v Vector3)Vector3{
	return NewVector3(s.x-v.X(),s.Y()-v.Y(),s.z-v.Z())
}

func (s vector3)Add(v Vector3)Vector3{
	return NewVector3(s.x+v.X(),s.y+v.Y(),s.z+v.Z())
}

func (v vector3)Mul(n float32)Vector3{
	return NewVector3(v.x*n,v.y*n,v.z*n)
}

func (v vector3)Div(n float32)Vector3{
	return NewVector3(v.x/n,v.y/n,v.z/n)
}

func (p vector3)Rotate(quaternion Quaternion)Vector3{
	ep := NewQuaternion(0,p.x,p.y,p.z)
	np := quaternion.Mul(ep).Mul(quaternion.Conjugate())
	return NewVector3(np.X(),np.Y(),np.Z())
}

func (left  vector3)Dot(right Vector3)float32{
	return left.X()*right.X()+left.Y()*right.Y()+left.Z()*right.Z()
}

func (left vector3)Cross(right Vector3)Vector3{
	return NewVector3(left.Y()*right.Z()-right.Y()*left.Z(),-(left.X()*right.Z()-right.X()*left.Z()),left.X()*right.Y()-right.X()*left.Y())
}

func (this vector3)Manitude()float32{
	return this.x*this.x+this.y*this.y+this.z*this.z
}

func (this vector3)SqrtManitude()float32{
	return float32(math.Sqrt(float64(this.Manitude())))
}

func (this vector3)Normalize()Vector3{
	l := this.SqrtManitude()
	res := NewVector3(this.X()/l,this.Y()/l,this.Z()/l)
	return res
}

func (this vector3)X()float32{
	return this.x
}

func (this vector3)Y()float32{
	return this.y
}

func (this vector3)Z()float32{
	return this.z
}
