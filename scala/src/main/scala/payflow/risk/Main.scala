package payflow.risk

object Main {
  def main(args: Array[String]): Unit = {
    val input = scala.io.Source.stdin.mkString.trim

    if (input.isEmpty) {
      System.err.println("no input provided on stdin")
      sys.exit(1)
    }

    try {
      val fields = Json.parseFlatJson(input)
      val charge = ChargeInput(
        id = fields.getOrElse("id", ""),
        amount = fields.getOrElse("amount", "0").toLong,
        currency = fields.getOrElse("currency", ""),
        customerId = fields.getOrElse("customer_id", "")
      )

      val (decision, reason) = RiskEngine.evaluate(charge)

      val output =
        s"""{"charge_id":"${Json.escape(charge.id)}","decision":"${Decision.toJsonValue(decision)}","reason":"${Json.escape(reason)}"}"""

      println(output)
    } catch {
      case e: Exception =>
        System.err.println(s"failed to evaluate charge: ${e.getMessage}")
        sys.exit(1)
    }
  }
}
