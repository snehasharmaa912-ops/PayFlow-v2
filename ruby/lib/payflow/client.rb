require "httparty"
require "json"

module Payflow
  class ApiError < StandardError; end

  class Client
    include HTTParty

    def initialize(base_url: ENV.fetch("PAYFLOW_API_URL", "http://localhost:8080"))
      self.class.base_uri base_url
    end

    def create_charge(amount:, currency:, customer_id:, idempotency_key:)
      response = self.class.post(
        "/charges",
        headers: { "Content-Type" => "application/json" },
        body: {
          amount: amount,
          currency: currency,
          customer_id: customer_id,
          idempotency_key: idempotency_key
        }.to_json
      )
      handle(response)
    end

    def get_charge(id)
      handle(self.class.get("/charges/#{id}"))
    end

    def list_charges
      handle(self.class.get("/charges"))
    end

    def ledger
      handle(self.class.get("/ledger"))
    end

    def verify_replay
      handle(self.class.get("/debug/verify-replay"))
    end

    private

    def handle(response)
      body = response.parsed_response
      unless response.success?
        message = body.is_a?(Hash) ? body["error"] : response.body
        raise ApiError, "#{response.code}: #{message}"
      end
      body
    end
  end
end
