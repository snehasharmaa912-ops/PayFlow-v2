package payflow.risk

object Json {

  def parseFlatJson(input: String): Map[String, String] = {
    val trimmed = input.trim.stripPrefix("{").stripSuffix("}")
    if (trimmed.trim.isEmpty) return Map.empty
    splitTopLevel(trimmed, ',').map { pair =>
      val idx = pair.indexOf(':')
      if (idx < 0) throw new IllegalArgumentException(s"malformed key:value pair: $pair")
      val rawKey = pair.substring(0, idx).trim
      val rawValue = pair.substring(idx + 1).trim
      val key = unquote(rawKey)
      val value = if (rawValue.startsWith("\"")) unquote(rawValue) else rawValue
      key -> value
    }.toMap
  }

  def escape(s: String): String =
    s.replace("\\", "\\\\").replace("\"", "\\\"")

  private def unquote(s: String): String = {
    val t = s.trim
    if (t.length >= 2 && t.startsWith("\"") && t.endsWith("\"")) {
      t.substring(1, t.length - 1).replace("\\\"", "\"").replace("\\\\", "\\")
    } else t
  }

  private def splitTopLevel(s: String, sep: Char): List[String] = {
    val buf = new StringBuilder
    var inQuotes = false
    val result = scala.collection.mutable.ListBuffer[String]()
    for (c <- s) {
      c match {
        case '"' =>
          inQuotes = !inQuotes
          buf += c
        case ch if ch == sep && !inQuotes =>
          result += buf.toString()
          buf.clear()
        case ch =>
          buf += ch
      }
    }
    result += buf.toString()
    result.toList.map(_.trim).filter(_.nonEmpty)
  }
}
