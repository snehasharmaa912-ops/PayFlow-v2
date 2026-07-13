package payflow.risk

class RiskEngineSpec extends munit.FunSuite {

  test("approves a normal charge") {
    val charge = ChargeInput("ch_1", 5000, "USD", "cus_1")
    val (decision, _) = RiskEngine.evaluate(charge)
    assertEquals(decision, Approve)
  }

  test("reviews a charge above the review threshold") {
    val charge = ChargeInput("ch_2", 150000, "USD", "cus_1")
    val (decision, _) = RiskEngine.evaluate(charge)
    assertEquals(decision, Review)
  }

  test("declines a charge above the decline threshold") {
    val charge = ChargeInput("ch_3", 2000000, "USD", "cus_1")
    val (decision, _) = RiskEngine.evaluate(charge)
    assertEquals(decision, Decline)
  }

  test("reviews an untrusted currency even at a small amount") {
    val charge = ChargeInput("ch_4", 500, "XYZ", "cus_1")
    val (decision, reason) = RiskEngine.evaluate(charge)
    assertEquals(decision, Review)
    assert(reason.contains("XYZ"))
  }

  test("decline threshold takes priority over currency check") {
    val charge = ChargeInput("ch_5", 2000000, "XYZ", "cus_1")
    val (decision, _) = RiskEngine.evaluate(charge)
    assertEquals(decision, Decline)
  }

  test("amount exactly at the review threshold is still approved") {
    val charge = ChargeInput("ch_6", RiskEngine.ReviewThresholdCents, "USD", "cus_1")
    val (decision, _) = RiskEngine.evaluate(charge)
    assertEquals(decision, Approve)
  }
}
