ThisBuild / scalaVersion := "2.13.14"

lazy val root = (project in file("."))
  .settings(
    name := "payflow-risk-engine",
    libraryDependencies += "org.scalameta" %% "munit" % "0.7.29" % Test,
    testFrameworks += new TestFramework("munit.Framework"),
    assembly / mainClass := Some("payflow.risk.Main"),
    assembly / assemblyJarName := "risk-engine.jar"
  )
