class AddTotalDifficulty < ActiveRecord::Migration
  def change
      add_column :block_headers, :td, :string, :limit => 32, null: false, :default => '0'
  end
end
