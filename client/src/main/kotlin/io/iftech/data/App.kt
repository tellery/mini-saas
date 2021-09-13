package io.iftech.data

import com.google.protobuf.Empty
import io.grpc.ManagedChannelBuilder
import io.grpc.Metadata
import io.grpc.stub.MetadataUtils
import io.iftech.data.user.UserServiceCoroutineGrpc
import io.ktor.application.*
import io.ktor.features.*
import io.ktor.http.*
import io.ktor.response.*
import io.ktor.routing.*
import io.ktor.serialization.*
import kotlinx.serialization.Serializable

fun Route.getUserRoute() {
    get("/user/{id}") {
        val id = call.parameters["id"] ?: return@get call.respondText(
            "Bad Request",
            status = HttpStatusCode.BadRequest
        )

        val port = System.getenv().getOrDefault("server.port", "9901").toInt()
        val channel = ManagedChannelBuilder
            .forAddress("envoy", port)
            .usePlaintext()
            .build()
        var stub = UserServiceCoroutineGrpc.newStub(channel)

        val header = Metadata()
        val key = Metadata.Key.of("User-Id", Metadata.ASCII_STRING_MARSHALLER)
        header.put(key, id)
        stub = MetadataUtils.attachHeaders(stub, header)

        val request = Empty.getDefaultInstance()
        val response = stub.getUserProfile(request)
        val user = User(
            id = id,
            name = response.name,
            age = response.age,
            city = response.city
        )
        call.respond(user)
    }
}

fun Application.registerUserRoutes() {
    routing { getUserRoute() }
}

fun Application.module() {
    install(ContentNegotiation) {
        json()
    }
    registerUserRoutes()
}

@Serializable
data class User(
    val id: String,
    val name: String,
    val age: String,
    val city: String
)

fun main(args: Array<String>): Unit = io.ktor.server.netty.EngineMain.main(args)
