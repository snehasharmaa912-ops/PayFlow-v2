package payflow.risk

class JsonSpec extends munit.FunSuite {

  test("parses a flat JSON object of strings and numbers") {
    val input = """{"id":"ch_1","amount":5000,"currency":"USD","customer_id":"cus_1"}"""
    val fields = Json.parseFlatJson(input)
    assertEquals(fields("id"), "ch_1")
    assertEquals(fields("amount"), "5000")
    assertEquals(fields("currency"), "USD")
    assertEquals(fields("customer_id"), "cus_1")
  }

  test("handles extra whitespace around fields") {
    val input = """{ "id" : "ch_2" , "amount" : 100 , "currency" : "EUR" , "customer_id" : "cus_2" }"""
    val fields = Json.parseFlatJson(input)
    assertEquals(fields("id"), "ch_2")
    assertEquals(fields("currency"), "EUR")
  }

  test("empty object parses to an empty map") {
    assertEquals(Json.parseFlatJson("{}"), Map.empty[String, String])
  }
}
