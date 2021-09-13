import com.google.protobuf.gradle.*
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

val protobufVersion: String by project
val krotoPlusVersion: String by project
val grpcVersion: String by project
val kotlinLoggingVersion: String by project
val logbackVersion: String by project

plugins {
    kotlin("jvm")
    id("com.google.protobuf")
    id("com.google.cloud.tools.jib")
    id("com.github.marcoferrer.kroto-plus")
}

dependencies {
    implementation(kotlin("stdlib-jdk8"))
    implementation("javax.annotation:javax.annotation-api:1.3.2")

    implementation("com.github.marcoferrer.krotoplus:kroto-plus-coroutines:$krotoPlusVersion")
    implementation("com.github.marcoferrer.krotoplus:kroto-plus-message:$krotoPlusVersion")
    implementation("com.github.marcoferrer.krotoplus:kroto-plus-test:$krotoPlusVersion")

    implementation("io.github.microutils:kotlin-logging:$kotlinLoggingVersion")
    implementation("ch.qos.logback:logback-classic:$logbackVersion")

    implementation("com.google.protobuf:protobuf-java:$protobufVersion")

    implementation("io.grpc:grpc-protobuf:$grpcVersion")
    implementation("io.grpc:grpc-stub:$grpcVersion")
    implementation("io.grpc:grpc-netty-shaded:$grpcVersion")
    implementation("io.grpc:grpc-services:$grpcVersion")
    implementation("io.grpc:grpc-kotlin-stub:1.1.0")
}

sourceSets.main {
    proto.srcDir("${rootProject.projectDir}/protobufs")
}

protobuf {
    protoc { artifact = "com.google.protobuf:protoc:$protobufVersion" }

    plugins {
        id("grpc") {
            artifact = "io.grpc:protoc-gen-grpc-java:$grpcVersion"
        }
        id("kroto") {
            artifact = "com.github.marcoferrer.krotoplus:protoc-gen-kroto-plus:$krotoPlusVersion"
        }
    }

    generateProtoTasks {
        val krotoConfig = file("${rootProject.projectDir}/krotoPlusConfig.json")

        all().forEach { task ->
            task.inputs.files(krotoConfig)
            task.plugins {
                id("grpc") {
                    outputSubDir = "java"
                }
                id("kroto") {
                    outputSubDir = "java"
                    option("ConfigPath=$krotoConfig")
                }
            }
        }
    }
}

// gRPC uses reflection function "setAccessible()" is disabled since jdk1.9
tasks.withType<KotlinCompile> {
    kotlinOptions {
        jvmTarget = "1.8"
        freeCompilerArgs = freeCompilerArgs + "-Xopt-in=kotlin.RequiresOptIn"
    }
}

tasks.withType<JavaCompile> {
    sourceCompatibility = "1.8"
    targetCompatibility = "1.8"
}

// config: https://github.com/GoogleContainerTools/jib/tree/master/jib-gradle-plugin#quickstart
jib {
    from {
        image = "springci/graalvm-ce:master-java8"
    }
    to {
        image = "mini-saas-server"
    }
    container {
        ports = listOf("8080")
    }
}
