package options

import "github.com/hydn-co/substrate/json/polymorphic"

func init() {
	polymorphic.RegisterType[ApplicationEntityCollectorOptions]()
	polymorphic.RegisterType[AccountEntityCollectorOptions]()
	polymorphic.RegisterType[GroupEntityCollectorOptions]()
	polymorphic.RegisterType[OwnerEntityCollectorOptions]()
	polymorphic.RegisterType[ApplicationRoleEntityCollectorOptions]()
}
