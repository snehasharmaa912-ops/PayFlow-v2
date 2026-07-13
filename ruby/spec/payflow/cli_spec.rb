require_relative "../spec_helper"
require_relative "../../lib/payflow/cli"

RSpec.describe Payflow::CLI do
  let(:base_url) { "http://localhost:8080" }

  before { ENV["PAYFLOW_API_URL"] = base_url }

  describe "#create" do
    it "prints the created charge" do
      stub_request(:post, "#{base_url}/charges")
        .to_return(
          status: 201,
          body: {
            id: "ch_1", amount: 5000, currency: "USD", customer_id: "cus_1",
            status: "succeeded", idempotency_key: "key-1", created_at: "2026-07-13T00:00:00Z"
          }.to_json
        )

      expect { Payflow::CLI.start(["create", "5000", "USD", "cus_1", "-k", "key-1"]) }
        .to output(/ID:\s+ch_1/).to_stdout
    end

    it "exits non-zero and prints an error on a validation failure" do
      stub_request(:post, "#{base_url}/charges")
        .to_return(status: 400, body: { error: "amount: must be a positive integer number of cents" }.to_json)

      expect {
        begin
          Payflow::CLI.start(["create", "-100", "USD", "cus_1"])
        rescue SystemExit
        end
      }.to output(/must be a positive integer/).to_stderr
    end
  end

  describe "#list" do
    it "prints a message when there are no charges" do
      stub_request(:get, "#{base_url}/charges").to_return(status: 200, body: "[]")

      expect { Payflow::CLI.start(["list"]) }.to output(/No charges yet/).to_stdout
    end
  end

  describe "#report" do
    it "prints balances and balanced status" do
      stub_request(:get, "#{base_url}/ledger").to_return(
        status: 200,
        body: { balances: { "platform:revenue" => 5000 }, balanced: true, sum: 0 }.to_json
      )

      expect { Payflow::CLI.start(["report"]) }.to output(/Balanced \(sum = 0\)/).to_stdout
    end
  end
end
