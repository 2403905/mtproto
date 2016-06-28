package mtproto

import(
  "fmt"
  "math/rand"
  // "os"
  // "regexp"
  // "strconv"
  // "strings"
)
type MTTeste struct{

}

func (o *MTTeste) EncodeVector(){
  fmt.Println("Running 2")
  contacts := make([]TL, 1)
  client_id := rand.Int63()
  phone := "+5581986272646"
  name := "Carlos"
  surname := "Lira"
  contacts[0] = TL_inputPhoneContact{
    client_id,
    phone,
    name,
    surname,
  }
  obj := contacts[0].encode()
  fmt.Println("Obj")
  y := NewEncodeBuf(256)
  y.Bytes(obj)
  y.dump()

  fmt.Println("Vector")
  x := NewEncodeBuf(256)
  x.Vector(contacts)
  x.dump()



  fmt.Println("Request")
  inputContacts := TL_contacts_importContacts{
    contacts,
    true,
  }
  req := inputContacts.encode()
  z := NewEncodeBuf(256)
  z.Bytes(req)
  z.dump()

}
