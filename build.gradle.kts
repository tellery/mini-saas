allprojects {
    group = "io.iftech.data"
    version = "1.0"
    repositories {
        // aliyun repository is preferred choice
        maven {
            url = uri("http://maven.aliyun.com/nexus/content/groups/public")
        }
        // jCenter is more popular than mavenCentral
        // ref: https://stackoverflow.com/questions/50726435/difference-among-mavencentral-jcenter-and-mavenlocal
        jcenter()
    }
    apply(plugin = "idea")
    apply(plugin = "distribution")
}

plugins {
    idea
    // Applying kotlin-jvm plugin with same version to subproject,
    // the version of the kotlin be declared in the settings.gradle.kts
    kotlin("jvm") apply false
}

idea {
    module {
        inheritOutputDirs = false
        outputDir = file("$buildDir/classes/kotlin/main")
        testOutputDir = file("$buildDir/classes/kotlin/test")
    }
}
