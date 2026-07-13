package payflow.risk

sealed trait Decision
case object Approve extends Decision
case object Review extends Decision
case object Decline extends Decision

object Decision {
  def toJsonValue(d: Decision): String = d match {
    case Approve => "approve"
    case Review  => "review"
    case Decline => "decline"
  }
}

final case class ChargeInput(id: String, amount: Long, currency: String, customerId: String)

object RiskEngine {

  val DeclineThresholdCents: Long = 1000000L // $10,000
  val ReviewThresholdCents: Long = 100000L   // $1,000
  val TrustedCurrencies: Set[String] = Set("USD", "EUR", "GBP", "CAD")

  def evaluate(charge: ChargeInput): (Decision, String) = charge match {
    case ChargeInput(_, amount, _, _) if amount > DeclineThresholdCents =>
      (Decline, s"amount $amount exceeds hard limit of $DeclineThresholdCents cents")

    case ChargeInput(_, amount, _, _) if amount > ReviewThresholdCents =>
      (Review, s"amount $amount exceeds review threshold of $ReviewThresholdCents cents")

    case ChargeInput(_, _, currency, _) if !TrustedCurrencies.contains(currency.toUpperCase) =>
      (Review, s"currency $currency is not in the trusted currency list")

    case _ =>
      (Approve, "within normal thresholds")
  }
}
