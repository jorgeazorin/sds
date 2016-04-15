package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
)

//Struct de los mensajes que se envian por el socket
type MensajeSocket struct {
	From          string   `json:"From"`
	To            int      `json:"To"`
	Password      string   `json:"Password"`
	Funcion       string   `json:"Funcion"`
	Datos         []string `json:"Datos"`
	Chat          int      `json:"Chat"`
	MensajeSocket string   `json:"MensajeSocket"`
}

type TodosLosDatos struct {
	Usuario Usuario  `json:"Usuario"`
	Claves  []string `json:"Claves"`
	Chats   []Chat   `json:"Chats"`
}

func ProcesarMensajeSocket(mensaje MensajeSocket, conexion net.Conn, usuario *Usuario) {

	//Para las operaciones con la BD
	var bd BD
	bd.username = "root"
	bd.password = ""
	bd.adress = ""
	bd.database = "sds"

	if mensaje.Funcion == "login" {

		//Rellenamos el usuario de la conexión con el login
		test := usuario.login(mensaje.From, mensaje.Password)

		//Si login incorrecto se lo decimos al cliente
		if test == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Login incorrecto"}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Si es correcto, aAñadimos la conexion al map de conexiones
		conexiones[usuario.Id] = conexion

		//Enviamos un mensaje de todo OK al usuario logeado
		mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Logeado correctamente"}
		EnviarMensajeSocketSocket(conexion, mesj)

	}

	if mensaje.Funcion == "enviar" {

		//Guardamos los mensajes en la BD
		var m Mensaje
		m.Texto = mensaje.MensajeSocket
		m.Idchat = 1
		m.Idemisor = usuario.Id
		m.Idclave = 1
		//bd.guardarMensajeBD(m)

		//Obtenemos los usuarios que pertenecen en el chat
		idChat := mensaje.Chat
		idusuarios := bd.getUsuariosChatBD(idChat)

		//Enviamos el mensaje a todos los usuarios de ese chat (incluido el emisor)
		for i := 0; i < len(idusuarios); i++ {
			conexion, ok := conexiones[idusuarios[i]]
			if ok {
				EnviarMensajeSocketSocket(conexion, mensaje)
			}
		}

	}

	if mensaje.Funcion == "obtenermensajeschat" {

		//Obtenemos los mensajes de ese chat
		idChat := mensaje.Chat

		//Comprobamos si ese usuario está en ese chat
		permitido := bd.usuarioEnChat(usuario.Id, idChat)

		if permitido == false {
			//Enviamos mensaje error
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "No perteneces al chat de estos mensajes."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Obtenemos los mensajes
		mensajes := bd.getMensajesChatBD(idChat)
		datos := make([]string, 0, 1)

		for i := 0; i < len(mensajes); i++ {
			fmt.Println("::::", mensajes[i].Id, mensajes[i].Texto)

			men := Mensaje{}
			men.Id = mensajes[i].Id
			men.Texto = mensajes[i].Texto

			//Codificamos los mensajes en json
			b, _ := json.Marshal(men)

			datos = append(datos, string(b))
		}

		//Enviamos los mensajes al usuario que los pidió
		mesj := MensajeSocket{From: usuario.Nombre, Datos: datos, MensajeSocket: "Mensajes recibidos:"}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Agregamos usuarios al chat
	if mensaje.Funcion == "agregarusuarioschat" {

		//Obtenemos los mensajes de ese chat
		idChat := mensaje.Chat

		//Comprobamos si ese usuario está en ese chat
		permitido := bd.usuarioEnChat(usuario.Id, idChat)

		if permitido == false {
			//Enviamos mensaje error
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "No tienes permiso para realizar esta acción, noperteneces al chat de estos mensajes."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		idusuarios := make([]int, 0, 1)
		for i := 0; i < len(mensaje.Datos); i++ {
			idusuario, _ := strconv.Atoi(mensaje.Datos[i])
			idusuarios = append(idusuarios, idusuario)
		}

		//Los agregamos llamando a la BD
		test := bd.addUsuariosChatBD(idChat, idusuarios)
		fmt.Println("Mira:", test)
		if test == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Hubo un error al realizar la operación."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Operación realizada correctamente."}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Eliminamos usuarios del chat
	if mensaje.Funcion == "eliminarusuarioschat" {

		//Obtenemos los mensajes de ese chat
		idChat := mensaje.Chat

		//Comprobamos si ese usuario está en ese chat
		permitido := bd.usuarioEnChat(usuario.Id, idChat)

		if permitido == false {
			//Enviamos mensaje error
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "No tienes permiso para realizar esta acción, noperteneces al chat de estos mensajes."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		idusuarios := make([]int, 0, 1)
		for i := 0; i < len(mensaje.Datos); i++ {
			idusuario, _ := strconv.Atoi(mensaje.Datos[i])
			idusuarios = append(idusuarios, idusuario)
		}

		//Los eliminamos llamando a la BD
		test := bd.removeUsuariosChatBD(idChat, idusuarios)

		if test == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Hubo un error al realizar la operación."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Operación realizada correctamente."}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Obtenemos clave pub de un usuario
	if mensaje.Funcion == "getclavepubusuario" {

		//Obtenemos id del usuario
		idusuario, _ := strconv.Atoi(mensaje.Datos[0])

		clavepub := bd.getClavePubUsuario(idusuario)

		if clavepub == "" {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al obtener clave del usuario."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, Datos: []string{clavepub}, MensajeSocket: "Clave pub usuario obtenida correctamente."}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Obtenemos clave cifrada de un mensaje
	if mensaje.Funcion == "getclavecifrarmensajechat" {

		//Obtenemos id del mensaje
		idchat, _ := strconv.Atoi(mensaje.Datos[0])

		prueba := bd.usuarioEnChat(usuario.Id, idchat)

		if prueba == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "No tienes permiso de acceso a los datos de este mensaje."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		clavemensaje, test := bd.getLastKeyMensaje(idchat, usuario.Id)

		if test == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al obtener clave para cifrar mensajes."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, Datos: []string{clavemensaje}, MensajeSocket: "Clave para mensajes obtenida correctamente."}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Crear una nueva clave para cifrar mensajes
	if mensaje.Funcion == "crearnuevoidparanuevaclavemensajes" {

		idclavemensajes := bd.CrearNuevaClaveMensajesBD()

		if idclavemensajes == 0 {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al crear id para nuevo conjunto de claves."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		cadena_idclavemensajes := strconv.FormatInt(idclavemensajes, 10)

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, Datos: []string{cadena_idclavemensajes}, MensajeSocket: "Id nuevo conjunto claves creado correctamente."}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Asocia nueva clave de un usuario con el id que indica ese nuevo conjunto de claves
	if mensaje.Funcion == "asociarnuevaclaveusuarioconidnuevoconjuntoclaves" {

		idconjuntoclaves, _ := strconv.Atoi(mensaje.Datos[0])
		claveusuario := mensaje.Datos[1]

		test := bd.GuardarClaveUsuarioMensajesBD(idconjuntoclaves, claveusuario, usuario.Id)

		if test == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al asociar la clave del usuario con el id del conjunto de claves."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Clave usuario asociada a id del conjunto de claves."}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	//Crea un usuario llamando a la BD
	if mensaje.Funcion == "registrarusuario" {

		var usuarionuevo Usuario

		usuarionuevo.Nombre = mensaje.Datos[0]
		usuarionuevo.Clavepubrsa = mensaje.Datos[1]
		usuarionuevo.Claveprivrsa = mensaje.Datos[2]
		usuarionuevo.Claveusuario = mensaje.Datos[3]

		if usuario.Id != 1 {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error, no tienes permiso para registrar a un usuario."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		test := bd.insertUsuarioBD(usuarionuevo)

		if test == false {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al intentar registrar al usuario."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}

		//Enviamos mensaje contestación
		mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Usuario registrado correctamente"}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	if mensaje.Funcion == "obtenerchats" {
		chats := bd.getChatsUsuarioBD(usuario.Id)

		datos := make([]string, 0, 1)

		for i := 0; i < len(chats); i++ {
			fmt.Println("::::", chats[i].Id, chats[i].Nombre)
			//Codificamos los mensajes en json
			b, _ := json.Marshal(chats[i])

			datos = append(datos, string(b))
		}

		//Enviamos los mensajes al usuario que los pidió
		mesj := MensajeSocket{From: usuario.Nombre, Datos: datos, MensajeSocket: "Chats:"}
		EnviarMensajeSocketSocket(conexion, mesj)

	}
	if mensaje.Funcion == "marcarmensajeleido" {
		i, _ := strconv.Atoi(mensaje.Datos[0])
		bd.marcarLeido(i)
	}

	if mensaje.Funcion == "obtenerclaves" {
		claves := bd.getClaves(usuario.Id)

		datos := make([]string, 0, 1)

		for i := 0; i < len(claves); i++ {
			fmt.Println("::::", claves[i])
			//Codificamos los mensajes en json

			datos = append(datos, claves[i])
		}

		//Enviamos los mensajes al usuario que los pidió
		mesj := MensajeSocket{From: usuario.Nombre, Datos: datos, MensajeSocket: "Claves:"}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	if mensaje.Funcion == "obtenerTodo" {
		todoslosdatos := TodosLosDatos{}
		todoslosdatos.Claves = bd.getClaves(usuario.Id)
		todoslosdatos.Usuario.Claveprivrsa = usuario.Claveprivrsa
		todoslosdatos.Usuario.Clavepubrsa = usuario.Clavepubrsa
		todoslosdatos.Usuario.Nombre = usuario.Nombre
		todoslosdatos.Chats = bd.getChatsUsuarioBD(usuario.Id)
		datos := make([]string, 0, 1)
		b, _ := json.Marshal(todoslosdatos)
		datos = append(datos, string(b))
		mesj := MensajeSocket{From: usuario.Nombre, Datos: datos, MensajeSocket: "Todos los datos:"}
		EnviarMensajeSocketSocket(conexion, mesj)
	}

	if mensaje.Funcion == "modificarusuario" {
		usuarioAux := Usuario{Id: usuario.Id, Nombre: usuario.Nombre, Claveprivrsa: mensaje.Datos[0], Clavepubrsa: mensaje.Datos[1], Claveusuario: mensaje.Datos[2]}
		boolean := bd.modificarUsuarioBD(usuarioAux)
		if boolean {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Usuario cambiado correctamente"}
			EnviarMensajeSocketSocket(conexion, mesj)
		} else {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al cambiar usuario"}
			EnviarMensajeSocketSocket(conexion, mesj)
		}
	}

	if mensaje.Funcion == "modificarchat" {
		i, err := strconv.Atoi(mensaje.Datos[0])

		if err != nil {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error con los parámetros recibidos."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}
		chat := Chat{Id: i, Nombre: mensaje.Datos[1]}
		//Comprobamos si ese usuario está en ese chat
		permitido := bd.usuarioEnChat(usuario.Id, chat.Id)

		if permitido == false {
			//Enviamos mensaje error
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "No tienes permiso para realizar esta acción, noperteneces al chat de estos mensajes."}
			EnviarMensajeSocketSocket(conexion, mesj)
			return
		}
		boolean := bd.modificarChatBD(chat)
		if boolean {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Chat cambiado correctamente"}
			EnviarMensajeSocketSocket(conexion, mesj)
		} else {
			mesj := MensajeSocket{From: usuario.Nombre, MensajeSocket: "Error al cambiar el chat"}
			EnviarMensajeSocketSocket(conexion, mesj)
		}
	}

}
