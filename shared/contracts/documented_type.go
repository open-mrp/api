package contracts

/*
An interface that types must implement to specify their documentation type.
*/
type DocumentedType interface {
	SchemaExample() any // Returns an example of the type
}
