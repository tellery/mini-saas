rootProject.name = "mini-saas"
include("server", "client")

pluginManagement {
    val kotlinVersion: String by settings
    val krotoPlusVersion: String by settings
    val protobufPluginVersion: String by settings
    val jibVersion: String by settings
    plugins {
        kotlin("jvm") version kotlinVersion
        kotlin("plugin.spring") version kotlinVersion
        id("com.github.marcoferrer.kroto-plus") version krotoPlusVersion
        id("com.google.protobuf") version protobufPluginVersion
        id("com.google.cloud.tools.jib") version jibVersion
    }
}
