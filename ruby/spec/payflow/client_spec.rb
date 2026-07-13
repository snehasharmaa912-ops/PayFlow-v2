require_relative "../spec_helper"

RSpec.describe Payflow::Client do
  let(:base_url) { "http://localhost:8080" }
  let(:client) { described_class.new(base_url: base_url) }

  describe "#create_charge" do
    it "posts the charge and returns the parsed response" do
      stub_request(:post, "#{base_url}/charges")
        .with(
          body: {
            amount: 5000,
            currency: "USD",
            customer_id: "cus_1",
            idempotency_key: "key-1"
          }.to_json
        )
        .to_return(
          status: 201,
          body: { id: "ch_1", amount: 5000, currency: "USD", customer_id: "cus_1", status: "succeeded" }.to_json,
          headers: { "Content-Type" => "application/json" }
        )

      result = client.create_charge(amount: 5000, currency: "USD", customer_id: "cus_1", idempotency_key: "key-1")

      expect(result["id"]).to eq("ch_1")
      expect(result["status"]).to eq("succeeded")
    end

    it "raises ApiError on a validation failure" do
      stub_request(:post, "#{base_url}/charges")
        .to_return(status: 400, body: { error: "amount: must be a positive integer number of cents" }.to_json)

      expect {
        client.create_charge(amount: -100, currency: "USD", customer_id: "cus_1", idempotency_key: "key-1")
      }.to raise_error(Payflow::ApiError, /must be a positive integer/)
    end
  end

  describe "#get_charge" do
    it "raises ApiError when the charge is not found" do
      stub_request(:get, "#{base_url}/charges/missing")
        .to_return(status: 404, body: { error: "charge not found" }.to_json)

      expect { client.get_charge("missing") }.to raise_error(Payflow::ApiError, /charge not found/)
    end
  end

  describe "#ledger" do
    it "returns balances and balanced status" do
      stub_request(:get, "#{base_url}/ledger")
        .to_return(
          status: 200,
          body: { balances: { "customer:cus_1" => -5000, "platform:revenue" => 5000 }, balanced: true, sum: 0 }.to_json
        )

      result = client.ledger
      expect(result["balanced"]).to eq(true)
      expect(result["balances"]["platform:revenue"]).to eq(5000)
    end
  end
end
