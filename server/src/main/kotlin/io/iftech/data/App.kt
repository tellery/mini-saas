package io.iftech.data

import com.google.protobuf.Empty
import io.grpc.*
import io.grpc.kotlin.CoroutineContextServerInterceptor
import io.grpc.util.TransmitStatusRuntimeExceptionInterceptor
import io.iftech.data.user.GetUserProfileResponse
import io.iftech.data.user.UserServiceCoroutineGrpc
import mu.KotlinLogging
import kotlin.coroutines.CoroutineContext
import kotlin.coroutines.coroutineContext

class UserService : UserServiceCoroutineGrpc.UserServiceImplBase() {
    companion object {
        val logger = KotlinLogging.logger { }
    }

    override suspend fun getUserProfile(request: Empty): GetUserProfileResponse {
        val id = System.getenv().getOrDefault("server.id", "1")

        logger.info("received message, server id: $id, header id: ${coroutineContext[UserIdElement]}")

        return when (id) {
            "1" -> GetUserProfileResponse {
                name = "Jack"
                age = "10"
                city = "Shanghai"
            }
            "2" -> GetUserProfileResponse {
                name = "Lily"
                age = "20"
                city = "Mianyang"
            }
            else -> throw StatusRuntimeException(Status.NOT_FOUND.withDescription("The id is not support in the system."))
        }
    }
}

class UserIdElement(val userIdString: String) : CoroutineContext.Element {
    companion object Key : CoroutineContext.Key<UserIdElement>
    override val key: CoroutineContext.Key<UserIdElement>
        get() = Key
}

object UserInterceptor : CoroutineContextServerInterceptor() {
    val logger = KotlinLogging.logger { }
    override fun coroutineContext(call: ServerCall<*, *>, headers: Metadata): CoroutineContext {
        val userIdString = headers.get(Metadata.Key.of("User-Id", Metadata.ASCII_STRING_MARSHALLER))
        logger.info("userId: $userIdString")
        return UserIdElement(userIdString!!)
    }
}

class RpcService {
    companion object {
        val logger = KotlinLogging.logger { }
    }

    private val port = System.getenv().getOrDefault("server.port", "8080").toInt()
    private val server = ServerBuilder.forPort(port)
        .addService(UserService())
        .intercept(UserInterceptor)
        .intercept(TransmitStatusRuntimeExceptionInterceptor.instance())
        .build()

    fun start() {
        server.start()
        logger.info("Server started, listening on {}", port)
        Runtime.getRuntime().addShutdownHook(
            Thread {
                logger.info("*** shutting down gRPC server since JVM is shutting down")
                this@RpcService.stop()
                logger.info("*** server shut down")
            }
        )
    }

    fun blockUntilShutdown() {
        server.awaitTermination()
    }

    private fun stop() {
        server.shutdown()
    }
}

fun main() {
    val app = RpcService()
    app.start()
    app.blockUntilShutdown()
}
