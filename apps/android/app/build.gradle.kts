plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "si.astrafoundry.arion"
    compileSdk = 36
    defaultConfig {
        applicationId = "si.astrafoundry.arion"
        // Android 13 adds checkRecognitionSupport, model downloads and
        // biasing strings for the on-device recognizer.
        minSdk = 33
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    buildFeatures {
        compose = true
    }
    sourceSets {
        getByName("main") {
            // The desktop's polishing recipe, bundled unchanged.
            assets.directories.add(layout.buildDirectory.dir("generated/recipe").get().asFile.path)
        }
    }
}

val copyRecipe by tasks.registering(Copy::class) {
    from(rootProject.file("../desktop/infrastructure/llm/prompt/format.json"))
    into(layout.buildDirectory.dir("generated/recipe"))
}
tasks.named("preBuild") { dependsOn(copyRecipe) }

dependencies {
    implementation(project(":core"))
    implementation(project(":design"))
    implementation(libs.activity.compose)
    implementation(libs.compose.icons)
    implementation(libs.lifecycle.runtime)
    implementation(libs.lifecycle.service)
    implementation(libs.savedstate)
    implementation(libs.coroutines.android)
    implementation(libs.genai.prompt)
    implementation(libs.serialization.json)
    debugImplementation(libs.compose.tooling)
}
