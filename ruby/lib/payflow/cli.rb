require "thor"
require "securerandom"
require "json"
require_relative "client"

module Payflow
  class CLI < Thor
    def initialize(*args)
      super
      @client = Client.new
    end

    desc "create AMOUNT CURRENCY CUSTOMER_ID", "Create a charge (amount in cents)"
    method_option :idempotency_key, aliases: "-k", desc: "Reuse a key to safely retry without double-charging"
    def create(amount, currency, customer_id)
      key = options[:idempotency_key] || SecureRandom.uuid
      charge = @client.create_charge(
        amount: amount.to_i,
        currency: currency,
        customer_id: customer_id,
        idempotency_key: key
      )
      print_charge(charge)
    rescue ApiError => e
      abort_with(e.message)
    end

    desc "list", "List all charges"
    def list
      charges = @client.list_charges
      if charges.empty?
        puts "No charges yet."
        return
      end
      charges.each { |c| print_charge_summary(c) }
    rescue ApiError => e
      abort_with(e.message)
    end

    desc "show CHARGE_ID", "Show a single charge"
    def show(charge_id)
      print_charge(@client.get_charge(charge_id))
    rescue ApiError => e
      abort_with(e.message)
    end

    desc "report", "Show ledger balances and whether the books are balanced"
    def report
      data = @client.ledger
      puts "Ledger balances:"
      data["balances"].each do |account, amount|
        puts "  #{account.ljust(24)} #{format_cents(amount)}"
      end
      puts ""
      puts data["balanced"] ? "Balanced (sum = 0)" : "NOT BALANCED (sum = #{data['sum']})"
    rescue ApiError => e
      abort_with(e.message)
    end

    desc "verify", "Replay the event log and compare it against live server state"
    def verify
      data = @client.verify_replay
      if data["matches"]
        puts "Replay matches live state."
        puts "  charges: #{data['live_charge_count']}"
        puts "  ledger balanced: #{data['live_balanced']}"
      else
        puts "MISMATCH between live state and replay:"
        puts JSON.pretty_generate(data)
      end
    rescue ApiError => e
      abort_with(e.message)
    end

    private

    def print_charge(charge)
      puts "ID:              #{charge['id']}"
      puts "Amount:          #{format_cents(charge['amount'])} #{charge['currency']}"
      puts "Customer:        #{charge['customer_id']}"
      puts "Status:          #{charge['status']}"
      puts "Risk decision:   #{charge['risk_decision']}" if charge["risk_decision"] && !charge["risk_decision"].empty?
      puts "Idempotency key: #{charge['idempotency_key']}"
      puts "Created:         #{charge['created_at']}"
    end

    def print_charge_summary(charge)
      puts "#{charge['id']}  #{format_cents(charge['amount']).rjust(12)} #{charge['currency']}  #{charge['status'].ljust(15)} #{charge['customer_id']}"
    end

    def format_cents(cents)
      format("%.2f", cents.to_f / 100)
    end

    def abort_with(message)
      warn "Error: #{message}"
      exit 1
    end
  end
end
