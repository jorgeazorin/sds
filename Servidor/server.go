package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
)

//Struct de conexión tiene el id del usuario y la conexión
type Conexion struct {
	conexion net.Conn
	usuario  int
}

/*
	Esto es la "Clase" de lo que va a ser el objeto c del main
	Tiene un map de conexiones y lo del mutex es para la memoria
	compartida entre los diferentes procesos

*/
type SafeCounter struct {
	v   map[Conexion]int
	mux sync.Mutex
}

/////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////

/*
	Función que guarda un socket en el map de conexiones y que se queda
	en un bucle infinito por si envia el cliente un mensaje
*/
func (c *SafeCounter) handleClientRead(conexion Conexion) {

	conn := conexion.conexion
	defer conn.Close()

	///////////////////////////////////
	//    Login      /////////////////
	//////////////////////////////////
	buf := make([]byte, 512)
	n, _ := conn.Read(buf)
	login := string(buf[:n])
	log.Printf("login: " + login)
	//esto es una especie de basura para probar
	if strings.Contains(login, "1") {
		conexion.usuario = 1
	} else {
		conexion.usuario = 2
	}

	///////////////////////////////////
	//    Añadimos al map la conexión con el usuario
	//////////////////////////////////

	//bloqueamos la memoria compartida
	c.mux.Lock()
	//La añadimos
	c.v[conexion]++
	//Y claro la debloqueamos
	c.mux.Unlock()
	//Enviamos un mensaje al cliente con ok como que el login se ha hecho correcto
	c.handleClientWrite(conexion.usuario, "OK")

	///////////////////////////////////
	//    Bucle infinito que lee cosas que envia el usuario
	//////////////////////////////////
	for {
		//Lee el mensaje
		n, _ := conn.Read(buf)
		//Envia el mensaje al usuario 2 (esto es para probar)
		c.handleClientWrite(2, string(buf[:n]))
	}
}

//FUncion que envia un mensaje a un cliente mediante un id y un string
func (c *SafeCounter) handleClientWrite(id int, s string) {
	var conexion Conexion
	//Bloqueamos la memoria compartida
	c.mux.Lock()
	//buscamos el socket del cliente al que enviar mensaje
	encontrado := false
	for k := range c.v {
		if k.usuario == id {
			conexion.conexion = k.conexion
			encontrado = true
			break
		}
	}
	c.mux.Unlock()
	//Si lo encontramos le enviamos el mensaje
	if encontrado {
		n, err := io.WriteString(conexion.conexion, s)
		if err != nil {
			log.Fatalf("client: write: %s", err)
		}
	}

}

/////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////
func main() {

	//Leer los ficheros de los certificados
	ca_b, _ := ioutil.ReadFile("ca.pem")
	ca, _ := x509.ParseCertificate(ca_b)
	priv_b, _ := ioutil.ReadFile("ca.key")
	priv, _ := x509.ParsePKCS1PrivateKey(priv_b)

	//Configurar los certificados en tls
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	cert := tls.Certificate{
		Certificate: [][]byte{ca_b},
		PrivateKey:  priv,
	}
	config := tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
	}
	config.Rand = rand.Reader

	//escuchar atodos
	service := "0.0.0.0:443"
	listener, err := tls.Listen("tcp", service, &config)
	if err != nil {
		log.Fatalf("server: listen: %s", err)
	}

	log.Print("server: listening")

	/*
		Esto es para paralelizar usando memoria compartida:
		se crea como un objeto c que tiene en su estructura un map de conexiones
		y ese objeto tiene las funciones de crear y enviar mensaje

		Nota para entender :
		Fíjate que solo va a haber un objeto c inicializado, este objeto tiene
		un map que es un vector de conexiones lo que hay son muchas ramas que
		ejecutan una función de c

	*/
	c := SafeCounter{v: make(map[Conexion]int)}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("server: accept: %s", err)
			break
		}
		defer conn.Close()
		log.Printf("server: accepted from %s", conn.RemoteAddr())

		//creamos una nueva conexión y se la pasamos al objeto c
		conexion := Conexion{}
		conexion.conexion = conn
		go c.handleClientRead(conexion)
	}
}
