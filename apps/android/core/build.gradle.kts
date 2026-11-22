plugins {
    alias(libs.plugins.kotlin.jvm)
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

dependencies {
    implementation(libs.serialization.json)
    api(libs.coroutines.core)
    testImplementation(libs.junit)
}

// The polishing recipe is the desktop's, read unchanged by the tests.
tasks.test {
    systemProperty("arion.recipe", rootProject.file("../desktop/infrastructure/llm/prompt/format.json").absolutePath)
}
