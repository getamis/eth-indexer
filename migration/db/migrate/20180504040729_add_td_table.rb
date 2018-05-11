class AddTdTable < ActiveRecord::Migration
  def change
      create_table :total_difficulty do |t|
        t.integer :block, :limit => 8, :null => false
        t.binary :hash, :limit => 32, :null => false
        t.string :td, null: false
      end
      add_index :total_difficulty, :hash, :unique => true
  end
end
